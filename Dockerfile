FROM alpine

ADD ./concurl /

EXPOSE 80
ENTRYPOINT ["/concurl"]

