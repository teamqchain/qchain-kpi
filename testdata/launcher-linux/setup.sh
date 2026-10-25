#!/bin/bash
# Runs as root inside the throwaway container: tools, an ordinary user, the
# repo at the VM's path, the fake docker first in PATH; then the driver.
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq >/dev/null
apt-get install -y -qq --no-install-recommends python3 git tmux iproute2 procps >/dev/null
useradd -m -s /bin/bash vmuser
mkdir -p /home/vmuser/Desktop/QChain/qchain-kpi /opt/fakebin
cp -a /t/repo/. /home/vmuser/Desktop/QChain/qchain-kpi/
install -m 755 /t/lt/fake-docker /opt/fakebin/docker
chown -R vmuser:vmuser /home/vmuser
echo "tools: $(python3 --version) $(git --version) $(tmux -V) bash $BASH_VERSION"
runuser -u vmuser -- env -i HOME=/home/vmuser USER=vmuser LOGNAME=vmuser TERM=xterm-256color LANG=C.UTF-8 \
  PATH=/opt/fakebin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin FAKE=/tmp/fake \
  bash /t/lt/driver.sh
