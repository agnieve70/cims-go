FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/cims ./cmd/cims

FROM alpine:3.22

WORKDIR /app
RUN adduser -D -h /app cims
COPY --from=build /out/cims /app/cims
COPY templates /app/templates
COPY static /app/static
COPY db/migrations /app/db/migrations
USER cims
EXPOSE 8080
CMD ["/app/cims"]
