FROM alpine:3.21.3 as urb
WORKDIR /dl
RUN apk update
RUN apk add curl
COPY install-urbit.sh /dl/urbit.sh
RUN chmod +x urbit.sh
RUN /bin/ash /dl/urbit.sh

FROM node:23-alpine3.20 AS ui
WORKDIR /src
ENV GODEBUG=http2client=0
ENV GOPROXY=https://proxy.golang.org,direct
COPY app/ui/ ./ui
WORKDIR /src/ui
RUN npm config set registry https://registry.npmjs.org/ && \
    npm config set fetch-retries 5 && \
    npm config set fetch-retry-mintimeout 20000 && \
    npm config set fetch-retry-maxtimeout 120000 && \
    npm ci --prefer-offline --no-audit
RUN npm ci
RUN npm run build

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
