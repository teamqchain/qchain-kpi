package main

// The connection to the network, as QChain's backend makes it
// (offchain/fabric.go getContract and chain.go withContract/chainSubmit at
// QChain commit 71a1db6): a new gRPC connection with TLS to the organisation's
// peer and a new Gateway for every transaction, the issuer identity of the
// General organisation, the same timeouts and receive limit, and
// SubmitTransaction's steps (endorse, submit, wait for the commit status),
// taken one by one here so that each is timed. fabric-gateway v1.7.0 and
// gRPC v1.68.0 are the backend's versions.

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// TxResult is one transaction, timed.
type TxResult struct {
	StartUnixNS int64
	ConnectNS   int64 // dial (gRPC connects lazily) and Gateway set-up
	EndorseNS   int64 // proposal, the TLS handshake on first use, endorsement by both organisations
	SubmitNS    int64 // until the orderer accepted the transaction
	CommitNS    int64 // until the peer reported the commit status
	TotalNS     int64 // start of the call to the commit status
	TxID        string
	Block       uint64
	Code        int32
	Valid       bool
}

// Ledger is what the runner needs from the network (a fake in the tests).
type Ledger interface {
	Put(ctx context.Context, id, payloadHex string) (TxResult, error)
	Block(ctx context.Context, n uint64) ([]byte, error)
	Get(ctx context.Context, id string) ([]byte, error)
	Close()
}

// GatewayConfig locates the peer and the identity.
type GatewayConfig struct {
	Endpoint   string // peer0.general.uae.com:9051
	ServerName string // peer0.general.uae.com
	TLSCACert  string // the peer's TLS CA certificate (PEM)
	MSPID      string // GeneralMSP
	MSPDir     string // issuer1's msp folder (signcerts/, keystore/)
	Channel    string // mychannel
	Chaincode  string // kpipayload
}

// DefaultGateway is QChain's issuer on the E14 network (paths in the material volume).
func DefaultGateway(root string) GatewayConfig {
	org := filepath.Join(root, "crypto-material/peerOrganizations/general.uae.com")
	return GatewayConfig{
		Endpoint:   "peer0.general.uae.com:9051",
		ServerName: "peer0.general.uae.com",
		TLSCACert:  filepath.Join(org, "peers/peer0.general.uae.com/tls/ca.crt"),
		MSPID:      "GeneralMSP",
		MSPDir:     filepath.Join(org, "users/issuer1@general.uae.com/msp"),
		Channel:    "mychannel",
		Chaincode:  "kpipayload",
	}
}

type gatewayLedger struct {
	cfg  GatewayConfig
	tls  credentials.TransportCredentials
	id   *identity.X509Identity
	sign identity.Sign
	// a long-lived connection for block and read-back queries (outside the timed path)
	conn *grpc.ClientConn
	gw   *client.Gateway
}

// NewGatewayLedger loads the identity and TLS material once.
func NewGatewayLedger(cfg GatewayConfig) (Ledger, error) {
	caPEM, err := os.ReadFile(cfg.TLSCACert)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("no certificate in %s", cfg.TLSCACert)
	}
	certPEM, err := readOne(filepath.Join(cfg.MSPDir, "signcerts"))
	if err != nil {
		return nil, err
	}
	keyPEM, err := readOne(filepath.Join(cfg.MSPDir, "keystore"))
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("cannot decode the issuer certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	id, err := identity.NewX509Identity(cfg.MSPID, cert)
	if err != nil {
		return nil, err
	}
	pk, err := identity.PrivateKeyFromPEM(keyPEM)
	if err != nil {
		return nil, err
	}
	sign, err := identity.NewPrivateKeySign(pk)
	if err != nil {
		return nil, err
	}
	l := &gatewayLedger{cfg: cfg, tls: credentials.NewClientTLSFromCert(pool, cfg.ServerName), id: id, sign: sign}
	if l.conn, l.gw, err = l.connect(); err != nil {
		return nil, err
	}
	return l, nil
}

// readOne returns the only file in dir (an msp's signcerts or keystore).
func readOne(dir string) ([]byte, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	if len(ents) != 1 {
		return nil, fmt.Errorf("%s holds %d files, want 1", dir, len(ents))
	}
	return os.ReadFile(filepath.Join(dir, ents[0].Name()))
}

// connect is the backend's getContract without the file reads (done once).
func (l *gatewayLedger) connect() (*grpc.ClientConn, *client.Gateway, error) {
	conn, err := grpc.Dial(l.cfg.Endpoint,
		grpc.WithTransportCredentials(l.tls),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(64<<20)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("grpc dial to %s: %w", l.cfg.Endpoint, err)
	}
	gw, err := client.Connect(l.id,
		client.WithSign(l.sign),
		client.WithClientConnection(conn),
		client.WithEvaluateTimeout(30*time.Second),
		client.WithEndorseTimeout(30*time.Second),
		client.WithSubmitTimeout(30*time.Second),
		client.WithCommitStatusTimeout(60*time.Second),
	)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	return conn, gw, nil
}

func (l *gatewayLedger) Put(ctx context.Context, id, payloadHex string) (TxResult, error) {
	r := TxResult{StartUnixNS: time.Now().UnixNano()}
	t0 := time.Now()
	conn, gw, err := l.connect()
	if err != nil {
		return r, fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()
	defer gw.Close()
	contract := gw.GetNetwork(l.cfg.Channel).GetContract(l.cfg.Chaincode)
	t1 := time.Now()
	proposal, err := contract.NewProposal("putPayload", client.WithArguments(id, payloadHex))
	if err != nil {
		return r, fmt.Errorf("proposal: %w", err)
	}
	tx, err := proposal.EndorseWithContext(ctx)
	if err != nil {
		return r, fmt.Errorf("endorse: %w", err)
	}
	t2 := time.Now()
	commit, err := tx.SubmitWithContext(ctx)
	if err != nil {
		return r, fmt.Errorf("submit: %w", err)
	}
	t3 := time.Now()
	status, err := commit.StatusWithContext(ctx)
	if err != nil {
		return r, fmt.Errorf("commit status: %w", err)
	}
	t4 := time.Now()
	r.ConnectNS = t1.Sub(t0).Nanoseconds()
	r.EndorseNS = t2.Sub(t1).Nanoseconds()
	r.SubmitNS = t3.Sub(t2).Nanoseconds()
	r.CommitNS = t4.Sub(t3).Nanoseconds()
	r.TotalNS = t4.Sub(t0).Nanoseconds()
	r.TxID = tx.TransactionID()
	r.Block = status.BlockNumber
	r.Code = int32(status.Code)
	r.Valid = status.Successful
	return r, nil
}

func (l *gatewayLedger) Block(ctx context.Context, n uint64) ([]byte, error) {
	qscc := l.gw.GetNetwork(l.cfg.Channel).GetContract("qscc")
	return qscc.EvaluateWithContext(ctx, "GetBlockByNumber", client.WithArguments(l.cfg.Channel, strconv.FormatUint(n, 10)))
}

func (l *gatewayLedger) Get(ctx context.Context, id string) ([]byte, error) {
	cc := l.gw.GetNetwork(l.cfg.Channel).GetContract(l.cfg.Chaincode)
	return cc.EvaluateWithContext(ctx, "getPayload", client.WithArguments(id))
}

func (l *gatewayLedger) Close() {
	l.gw.Close()
	l.conn.Close()
}
