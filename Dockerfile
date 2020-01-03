FROM golang:1.13.5
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o udp-procfs-exporter ./main.go
RUN go build -o simple-server ./simple-server.go
CMD ["./udp-procfs-exporter"]