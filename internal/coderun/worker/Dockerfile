FROM golang:1.19

WORKDIR /worker_dir

# Download dependencies.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy application source.
COPY . .

# Build executable
RUN go build -o worker internal/coderun/worker/main.go

CMD ["./worker"]
