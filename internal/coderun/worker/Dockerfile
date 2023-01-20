FROM golang:1.19

WORKDIR /worker_dir

# Download dependencies.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy application source.
COPY . .

# Build executable
RUN go build -o worker internal/coderun/worker/main.go

# Install python3
RUN apt-get update
RUN yes | apt-get install python3 python3-pip

# Install python3 requirements
RUN pip3 install -r internal/coderun/worker/requirements.txt

CMD ["./worker"]
