```mermaid

graph LR
    subgraph "Load Generator"
        TC[Test Script<br/>10,000 messages<br/>Burst: 100 msg batches]
    end
    
    subgraph "Producer"
        OS[Order Service<br/>linger.ms=0<br/>batch.size=16KB<br/>compression=none]
    end
    
    subgraph "Kafka Cluster"
        K[Redpanda<br/>3 brokers<br/>6 partitions]
    end
    
    subgraph "Consumer"
        IC[Inventory Consumer<br/>Sequential processing<br/>fetch.min.bytes=1]
    end
    
    subgraph "Database"
        DB[(PostgreSQL<br/>Individual INSERTs)]
    end
    
    TC -->|HTTP POST<br/>21s duration| OS
    OS -->|10,000 individual<br/>produce() calls| K
    K -->|Poll<br/>1 msg at a time| IC
    IC -->|INSERT<br/>per message| DB
    
    style TC fill:#e1f5ff
    style OS fill:#ffcccc
    style K fill:#ffe1e1
    style IC fill:#ffcccc
    style DB fill:#f0e1ff

```