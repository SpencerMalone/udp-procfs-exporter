./simple-server &
sleep 1
./udp-procfs-exporter simple-server 8125
