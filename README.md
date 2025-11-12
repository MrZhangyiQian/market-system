# Market System 文档
Market System ！
适合新兴交易所和新手熟悉行情系统业务！同时支持内部交易数据和外部数据源融合。

- 当前版本：v1.0
- 最后更新：2025-11-11
- Go 版本：1.21+
- go-zero 版本：1.6.1

##  核心特性

- 微服务架构（基于 go-zero）
- 高性能实时数据采集
- 多交易所支持
- 混合模式（内外部数据融合）
- K线实时聚合
- WebSocket 推送
- RESTful API
- Docker 部署

market-system/
    common/                 # 公共库
        config/             # 配置结构
        models/             # 数据模型
        utils/              # 工具函数
        constants/          # 常量定义
    configs/                # 配置文件
    deploy/                 # 部署文件
    services/
        collector/          # 采集服务
          cmd/              # 入口
          internal/         # 内部实现
             adapters/      # 交易所适配器
             publisher/     # Kafka 发布者
         processor/         # 处理服务
           cmd/
           internal/
             consumer/      # Kafka 消费者
             handler/       # 数据处理
             storage/       # 存储层
        api/                # API 服务
           cmd/
            internal/
               config/      # 配置结构          
               handler/     # HTTP Handler
               logic/       # 业务逻辑
               svc/         # 服务上下文
               types/       # 业务类型
   
