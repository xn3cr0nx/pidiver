# ############################
# # STEP 1 build executable binary
# ############################
FROM golang as builder

# # Install git + SSL ca certificates.
# # Git is required for fetching the dependencies.
# # Ca-certificates is required to call HTTPS endpoints.
# RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

# # Create appuser
# RUN adduser -D -g '' appuser

WORKDIR /app/pidiver/
COPY . .

# # Fetch dependencies.

# # Using go mod with go 1.11
RUN go mod download

WORKDIR /app/pidiver/server

# # Build the binary
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o server
# RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o $GOPATH/src/pidiver/

EXPOSE 14265

CMD ["./server"]

# # ############################
# # # STEP 2 build a small image
# # ############################
# FROM scratch

# # # Import from builder.
# # COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# # COPY --from=builder /etc/passwd /etc/passwd

# # # Copy our static executable
# COPY --from=builder /app/pidiver/server/server /app/pidiver/server/server

# # # Use an unprivileged user.
# # USER appuser

# EXPOSE 14265

# # # Run the hello binary.
# CMD [ "/app/pidiver/server/server" ]