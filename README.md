# udp-procfs-exporter

To test this:
1) run ./script/test

To run this:
1) Build ./script/build
2) Run ./udp-procfs-exporter <proc name to watch>. Note: Your proc name may be shortened by procfs to a max of 15 characters. Ex: to watch `udp-procfs-exporter`, you need to use `udp-procfs-expo` as the argument