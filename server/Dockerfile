# Anmo server — SQLite 存储版（2026-09-28 起，MySQL 退役）。
# modernc.org/sqlite 为纯 Go 驱动：CGO_ENABLED=0 静态编译，无 libc 依赖。
# yun1 部署流程不变：WSL 构建镜像 → save|scp|load → force-recreate（内存红线）。
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/anmo ./cmd/anmo

FROM alpine:3.20
RUN adduser -D -u 10001 anmo \
 && mkdir -p /data /backups && chown -R anmo:anmo /data /backups
WORKDIR /app
COPY --from=build /out/anmo /app/anmo
COPY migrations /app/migrations
# SQLite 数据文件放 /data（compose 挂卷）；备份挂 /backups。
# 绑定挂载宿主机目录时需 chown 10001（yun1 部署脚本已处理）。
VOLUME ["/data", "/backups"]
USER anmo
ENV ANMO_DB_PATH=/data/anmo.db
EXPOSE 8080
ENTRYPOINT ["/app/anmo"]
