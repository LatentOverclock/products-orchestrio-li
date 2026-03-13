FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN /usr/local/go/bin/go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux /usr/local/go/bin/go build -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
