FROM golang:1.19-alpine
# TODO: multi-stage build to minify container size
WORKDIR /logtopus
COPY . .
RUN go mod download

RUN go build -o ./logtopus ./cmd/server/main.go
RUN chmod +x ./logtopus
RUN ls -la .
EXPOSE 5000

CMD [ "/logtopus/logtopus", "/logtopus/configs/deploy.env" ]