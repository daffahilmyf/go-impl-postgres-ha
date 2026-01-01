FROM golang:1.25-alpine AS build

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/app main.go

FROM gcr.io/distroless/static:nonroot

WORKDIR /app
COPY --from=build /out/app /app/app

USER nonroot:nonroot
ENTRYPOINT ["/app/app"]
