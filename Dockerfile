FROM golang:1.21.4

# Set destination for COPY
WORKDIR /mailing

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code.
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o ./.bin/mailing ./cmd/mailing/main.go

# Optional:
# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/engine/reference/builder/#expose
EXPOSE 8080

# Run
CMD ["./.bin/mailing"]

