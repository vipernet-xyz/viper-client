FROM golang:1.22-alpine

# Install Git and other dependencies
RUN apk add --no-cache git

WORKDIR /app

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Install Air for hot reloading
RUN go install github.com/cosmtrek/air@v1.42.0

EXPOSE 8080

# Start Air for hot reloading
CMD ["air"] 