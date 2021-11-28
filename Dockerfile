FROM golang:alpine as builder

# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Move to working directory /build
WORKDIR /build

# Copy the code into the container
COPY . .

# Download go modules
RUN go mod init mappin-server && go get

# Build the application
RUN go build -o app .



FROM alpine:latest

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from the builder
COPY --from=builder /build/app .

# Create the logs folder
RUN mkdir logs

# Export necessary port
EXPOSE 8080

# Command to run when starting the container
CMD ["/dist/app"]
