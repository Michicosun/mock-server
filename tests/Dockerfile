FROM golang:1.19

WORKDIR /tests_dir

# Copy application source.
COPY . .

# Download dependencies.
RUN go mod download && go mod verify

# Build executable
RUN go build ./...

CMD [ "go", "test", "-v", "./..." ]
