FROM centos
ADD ./main/main /root/p2p/main
EXPOSE 54321/udp
EXPOSE 3478/udp
ENTRYPOINT ["/root/p2p/main"]

