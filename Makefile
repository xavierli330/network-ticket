.PHONY: help deploy start stop restart status logs backup uninstall dev-backend dev-frontend build

help:
	@echo "网络工单平台 - 快捷命令"
	@echo ""
	@echo "部署与管理:"
	@echo "  make deploy      一键部署"
	@echo "  make start       启动服务"
	@echo "  make stop        停止服务"
	@echo "  make restart     重启服务"
	@echo "  make status      查看服务状态"
	@echo "  make logs        查看实时日志"
	@echo "  make backup      备份数据库"
	@echo "  make uninstall   完全卸载"
	@echo ""
	@echo "开发:"
	@echo "  make dev-backend  启动后端热重载开发"
	@echo "  make dev-frontend 启动前端开发服务器"
	@echo "  make build        编译后端"
	@echo ""

deploy:
	./deploy.sh

start:
	./manage.sh start

stop:
	./manage.sh stop

restart:
	./manage.sh restart

status:
	./manage.sh status

logs:
	./manage.sh logs

backup:
	./manage.sh backup

uninstall:
	./manage.sh uninstall

update:
	./manage.sh update

dev-backend:
	cd backend && make dev

dev-frontend:
	@echo "cd frontend && npm run dev"

build:
	cd backend && make build
