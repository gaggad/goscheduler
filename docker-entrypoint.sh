#!/bin/sh

# 如果没有参数，默认使用web模式
if [ -z "$1" ]; then
    exec /app/goscheduler web
fi

case "$1" in
    "web")
        exec /app/goscheduler web
        ;;
    "node")
        shift
        exec /app/goscheduler-node "$@"
        ;;
    *)
        echo "Usage: $0 [web|node] [args...]"
        echo "  web: Run goscheduler web server"
        echo "  node: Run goscheduler-node with additional arguments"
        exit 1
        ;;
esac