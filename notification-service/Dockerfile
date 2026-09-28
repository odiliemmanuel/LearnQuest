FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /notification-service .

FROM gcr.io/distroless/static-debian12
COPY --from=build /notification-service /notification-service
USER nonroot:nonroot
ENTRYPOINT ["/notification-service"]
