FROM golang:1.18

WORKDIR /usr/src/app

ENV AWS_ID=""
ENV AWS_SECRET=""
ENV DOCKER_APP_ID=""
ENV DOCKER_GUILD_ID=""
ENV DOCKER_CHANNEL_ID=""
ENV DOCKER_ROLE_ID=""
ENV DOCKER_TOKEN=""

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN go build -v -o app
RUN chmod +x start.sh

CMD ["./start.sh"]