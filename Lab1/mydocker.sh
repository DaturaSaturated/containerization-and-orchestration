#!/bin/bash

sudo ip link delete veth-host 2>/dev/null || true

CGNAME="mydocker-$$"
sudo mkdir -p /sys/fs/cgroup/$CGNAME

echo 100M | sudo tee /sys/fs/cgroup/$CGNAME/memory.max
echo "50000 100000" | sudo tee /sys/fs/cgroup/$CGNAME/cpu.max
echo 30 | sudo tee /sys/fs/cgroup/$CGNAME/pids.max

(
    MYPID=$BASHPID
    echo $MYPID | sudo tee /sys/fs/cgroup/$CGNAME/cgroup.procs > /dev/null
        exec unshare --pid --fork --mount --mount-proc --uts --ipc --net --user --map-root-user --cgroup \
      -- sh -c 'ip link set lo up && capsh --drop=all -- -c "./seccomp ./api"'
) &

APP_PID=$!

sleep 1

sudo ip link add veth-host type veth peer name veth-cont
sudo ip link set veth-cont netns $APP_PID
sudo ip addr add 10.0.0.1/24 dev veth-host
sudo ip link set veth-host up
sudo nsenter -t $APP_PID -n ip addr add 10.0.0.2/24 dev veth-cont
sudo nsenter -t $APP_PID -n ip link set veth-cont up

wait $APP_PID

sudo ip link delete veth-host
sudo rmdir /sys/fs/cgroup/$CGNAME


