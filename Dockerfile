FROM golang:1.22-alpine

ENV USER=tiamat
ENV GROUPNAME=$USER
ENV UID=1000
ENV GID=1000
ENV APP_HOME="/home/tiamat/app"

RUN addgroup --gid "$GID" "$GROUPNAME"
RUN adduser --gecos "" --disabled-password --ingroup "$GROUPNAME" --uid "$UID" "$USER"

USER "$USER"

RUN mkdir "$APP_HOME"

ADD . "$APP_HOME"

WORKDIR "$APP_HOME"

RUN go build -o main .

CMD ["./main", "server"]
