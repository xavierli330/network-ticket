#!/bin/bash

set -e

PROJECT_NAME="network-ticket"
COMPOSE_FILE="docker-compose.yaml"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

DOCKER_COMPOSE="docker compose"
if command -v docker-compose >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker-compose"
fi

usage() {
    echo "网络工单平台 - 管理脚本"
    echo ""
    echo "用法: $0 <命令>"
    echo ""
    echo "命令:"
    echo "  deploy      一键部署（首次安装）"
    echo "  start       启动服务"
    echo "  stop        停止服务"
    echo "  restart     重启服务（不重新创建容器）"
    echo "  reload      重新加载配置（修改 .env 后使用）"
    echo "  status      查看服务状态"
    echo "  logs        查看实时日志"
    echo "  update      更新代码后重新构建部署"
    echo "  backup      备份数据库"
    echo "  uninstall   完全卸载（删除容器与数据卷）"
    echo ""
}

cmd_start() {
    echo "正在启动服务..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE start
    echo -e "${GREEN}✓ 服务已启动${NC}"
    echo "  访问地址: http://localhost"
}

cmd_stop() {
    echo "正在停止服务..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE stop
    echo -e "${GREEN}✓ 服务已停止${NC}"
}

cmd_restart() {
    echo "正在重启服务..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE restart
    echo -e "${GREEN}✓ 服务已重启${NC}"
    echo "  访问地址: http://localhost"
}

cmd_reload() {
    echo "正在重新加载配置并启动服务..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE up -d --force-recreate
    echo -e "${GREEN}✓ 配置已重新加载${NC}"
    echo "  访问地址: http://localhost"
}

cmd_status() {
    echo "服务状态:"
    $DOCKER_COMPOSE -f $COMPOSE_FILE ps
}

cmd_logs() {
    if [ -n "$2" ]; then
        $DOCKER_COMPOSE -f $COMPOSE_FILE logs -f "$2"
    else
        $DOCKER_COMPOSE -f $COMPOSE_FILE logs -f
    fi
}

cmd_update() {
    echo "正在重新构建并部署..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE down
    $DOCKER_COMPOSE -f $COMPOSE_FILE up -d --build
    $DOCKER_COMPOSE -f $COMPOSE_FILE restart nginx
    echo -e "${GREEN}✓ 更新完成${NC}"
}

cmd_backup() {
    BACKUP_DIR="./backups"
    mkdir -p "$BACKUP_DIR"
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BACKUP_FILE="$BACKUP_DIR/network_ticket_${TIMESTAMP}.sql"

    # 读取 .env 中的 MySQL root 密码
    if [ -f ".env" ]; then
        MYSQL_ROOT_PASSWORD=$(grep "^MYSQL_ROOT_PASSWORD=" .env | cut -d= -f2-)
    fi
    MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD:-root_password}

    echo "正在备份数据库..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE exec -T mysql mysqldump \
        -u root -p"$MYSQL_ROOT_PASSWORD" \
        --databases network_ticket \
        --single-transaction \
        > "$BACKUP_FILE"

    echo -e "${GREEN}✓ 备份完成: $BACKUP_FILE${NC}"
}

cmd_uninstall() {
    echo -e "${RED}警告：此操作将删除所有容器和数据卷，数据将不可恢复！${NC}"
    echo ""
    read -p "请输入 'yes' 确认卸载: " confirm
    if [ "$confirm" != "yes" ]; then
        echo "已取消卸载"
        exit 0
    fi

    echo "正在卸载..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE down -v --remove-orphans

    # 可选：清理构建缓存
    read -p "是否清理 Docker 构建缓存? [y/N] " clean_cache
    if [ "$clean_cache" = "y" ] || [ "$clean_cache" = "Y" ]; then
        docker system prune -f
    fi

    echo -e "${GREEN}✓ 卸载完成${NC}"
}

cmd_deploy() {
    exec ./deploy.sh
}

# 主逻辑
COMMAND="${1:-}"

if [ -z "$COMMAND" ]; then
    usage
    exit 0
fi

case "$COMMAND" in
    deploy)
        cmd_deploy
        ;;
    start)
        cmd_start
        ;;
    stop)
        cmd_stop
        ;;
    restart)
        cmd_restart
        ;;
    reload)
        cmd_reload
        ;;
    status)
        cmd_status
        ;;
    logs)
        cmd_logs "$@"
        ;;
    update)
        cmd_update
        ;;
    backup)
        cmd_backup
        ;;
    uninstall)
        cmd_uninstall
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        echo -e "${RED}未知命令: $COMMAND${NC}"
        usage
        exit 1
        ;;
esac
