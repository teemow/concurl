FROM flynn/busybox

ADD ./concurl /

EXPOSE 80
ENTRYPOINT ["/concurl"]

