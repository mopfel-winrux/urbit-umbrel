
FROM alpine:3.21.3 as urb
WORKDIR /dl
RUN apk update
RUN apk add curl
COPY install-urbit.sh /dl/urbit.sh
RUN chmod +x urbit.sh
RUN /bin/ash /dl/urbit.sh

FROM node:23-alpine3.20 AS ui
WORKDIR /src
COPY app/ui/ ./ui
RUN cd ui && npm ci --silent && npm run build --silent

FROM golang:1.24.2-alpine3.21 AS builder
WORKDIR /app
COPY app/go.* ./
RUN go mod download
COPY app/ .
COPY --from=ui /src/ui/dist /app/ui/dist
RUN CGO_ENABLED=0 go build -o /server .

FROM alpine:3.21.3
COPY --from=builder /server /server
COPY --from=urb /usr/sbin/urbit /usr/bin/urbit
EXPOSE 8080
ENTRYPOINT ["/server"]