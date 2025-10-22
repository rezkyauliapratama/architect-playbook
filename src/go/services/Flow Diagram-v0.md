```mermaid
    graph LR
        %% Nodes
        CLIENT[Client]
        PAY[Payment Gateway Service]
        ACC[Account Service]
        LED[Ledger Service]
        FEE[Fee Service]
        BANK[External Bank / BI-FAST Adapter]
        NOTIF[Notification Service]
        RECON[Reconciliation Service]
        REPORT[Reporting Service]

        %% Edges with labels (order matters for linkStyle indices)
        CLIENT -- "REST/gRPC Request\nUser Transaction" --> PAY
        PAY -- "API Call\nHold / Debit / Credit" --> ACC
        PAY -- "API Call\nPost Double-entry Journal" --> LED
        PAY -- "API Call\nCalculate Fees" --> FEE
        PAY -- "Async Event\nSend Transaction Notification" --> NOTIF
        PAY -- "API Call\nTransfer Request / Status Inquiry" --> BANK
        RECON -- "Scheduled Job\nFetch External Report" --> BANK
        RECON -- "Batch Compare\nValidate Account Balances" --> ACC
        RECON -- "Batch Compare\nReconcile Ledger Entries" --> LED
        REPORT -- "Event Stream / Read Model\nConsume Ledger Events" --> LED

        %% Link styles (indices 0..9 match the edges above)
        linkStyle 0 stroke:#2ecc71, stroke-width:3px;
        linkStyle 1 stroke:#27ae60, stroke-width:2.5px;
        linkStyle 2 stroke:#27ae60, stroke-width:2.5px;
        linkStyle 3 stroke:#27ae60, stroke-width:2.5px;
        linkStyle 4 stroke:#9b59b6, stroke-width:3px;
        linkStyle 5 stroke:#2980b9, stroke-width:3px;
        linkStyle 6 stroke:#e67e22, stroke-width:3px;
        linkStyle 7 stroke:#e67e22, stroke-width:3px;
        linkStyle 8 stroke:#e67e22, stroke-width:3px;
        linkStyle 9 stroke:#3498db, stroke-width:3px;

        %% Optional simple node styling (most renderers accept this)
        classDef core fill:#2c3e50,stroke:#1abc9c,color:#ecf0f1;
        class CLIENT,PAY,ACC,LED,FEE,NOTIF,BANK,RECON,REPORT core;
