#FROM debian
#COPY cluster-autoscaler /app
#ENTRYPOINT /app

FROM alpine
RUN apk update && apk add ca-certificates
RUN apk add --update tzdata
ENV TZ=America/Edmonton
RUN rm -rf /var/cache/apk/*
COPY cluster-autoscaler /app
ENTRYPOINT /app