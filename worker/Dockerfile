FROM golang:1.16 as build
LABEL stage=intermediate
WORKDIR /app
COPY go.mod .
COPY go.sum .
COPY worker.go .
ENV CGO_ENABLED=0
RUN go build .

FROM scratch
WORKDIR /app
COPY --from=build /app/worker /app
ENTRYPOINT ["/app/worker"]
CMD ["/app/worker"]
