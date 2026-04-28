#!/bin/bash
set -e

PROJECT_NAME="network-ticket"
COMPOSE_FILE="docker-compose.yaml"

echo "============================================"
echo "  网络工单平台 - 一键部署脚本"
echo "============================================"
echo ""

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查命令
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# 1. 检查 Docker
if ! command_exists docker; then
    echo -e "${RED}错误：未检测到 Docker，请先安装 Docker${NC}"
    echo "  安装指南: https://docs.docker.com/get-docker/"
    exit 1
fi

if ! command_exists docker-compose && ! docker compose version >/dev/null 2>&1; then
    echo -e "${RED}错误：未检测到 Docker Compose，请先安装${NC}"
    echo "  安装指南: https://docs.docker.com/compose/install/"
    exit 1
fi

DOCKER_COMPOSE="docker compose"
if command_exists docker-compose; then
    DOCKER_COMPOSE="docker-compose"
fi

echo -e "${GREEN}✓ Docker 环境已就绪${NC}"

# 2. 检查环境变量文件
if [ ! -f ".env" ]; then
    echo ""
    echo "首次部署，正在生成环境变量配置..."
    cp .env.example .env
    echo -e "${GREEN}✓ 已创建 .env${NC}"
    echo -e "${YELLOW}  提示：生产环境请编辑 .env 修改数据库密码和 JWT Secret${NC}"
else
    echo -e "${GREEN}✓ 环境变量配置已存在${NC}"
fi

# 3. 检查后端配置文件
if [ ! -f "backend/config.yaml" ]; then
    echo ""
    echo "首次部署，正在生成后端配置文件..."
    cp backend/config.example.yaml backend/config.yaml
    echo -e "${GREEN}✓ 已创建 backend/config.yaml${NC}"
else
    echo -e "${GREEN}✓ 后端配置文件已存在${NC}"
fi

# 4. 构建并启动
echo ""
echo "正在构建并启动服务，这可能需要几分钟..."
$DOCKER_COMPOSE -f $COMPOSE_FILE up -d --build

# 5. 等待 MySQL 就绪
echo ""
echo "等待数据库就绪..."
for i in {1..30}; do
    if $DOCKER_COMPOSE -f $COMPOSE_FILE ps mysql | grep -q "healthy"; then
        echo -e "${GREEN}✓ MySQL 已就绪${NC}"
        break
    fi
    sleep 2
    echo -n "."
done

# 6. 执行数据库迁移
echo ""
echo "正在执行数据库迁移..."
MYSQL_ROOT_PASSWORD=$(grep "^MYSQL_ROOT_PASSWORD=" .env | cut -d= -f2-)
MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD:-root_password}
docker run --rm --network "${PROJECT_NAME}_default" \
    -v "$(pwd)/backend/migrations:/migrations" \
    migrate/migrate:latest \
    -path /migrations \
    -database "mysql://root:${MYSQL_ROOT_PASSWORD}@tcp(mysql:3306)/network_ticket?multiStatements=true" \
    up
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 数据库迁移完成${NC}"
else
    echo -e "${YELLOW}⚠ 数据库迁移可能已执行过或出现错误，继续启动...${NC}"
fi

# 7. 等待后端就绪
echo ""
echo "等待后端服务就绪..."
for i in {1..30}; do
    if curl -s http://localhost:8080/api/v1/auth/login -o /dev/null -w "%{http_code}" | grep -Eq "400|200|401|404"; then
        echo -e "${GREEN}✓ 后端服务已就绪${NC}"
        break
    fi
    sleep 2
    echo -n "."
done

echo ""
echo "============================================"
echo -e "${GREEN}  部署完成！${NC}"
echo "============================================"
echo ""
echo "  访问地址: http://localhost"
echo ""
echo "  默认管理员账号:"
echo "    用户名: admin"
echo "    密码:   admin123"
echo ""
echo -e "  ${YELLOW}⚠  首次登录后请立即修改默认密码${NC}"
echo ""
echo "  常用管理命令:"
echo "    ./manage.sh stop      停止服务"
echo "    ./manage.sh start     启动服务"
echo "    ./manage.sh restart   重启服务"
echo "    ./manage.sh reload    重新加载配置"
echo "    ./manage.sh status    查看状态"
echo "    ./manage.sh logs      查看日志"
echo "    ./manage.sh backup    备份数据库"
echo "    ./manage.sh uninstall 完全卸载（含数据）"
echo ""
