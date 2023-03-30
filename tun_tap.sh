sudo ip tuntap add mode tun dev tun0
sudo ip link set tun0 up
sudo ip addr add 10.0.0.1/24 dev tun0
