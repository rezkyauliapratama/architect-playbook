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
            
            %% Edges with labels (order matters for linkStyle indices)
            CLIENT e1@-- "API Call User Transaction" --> PAY
            PAY e2@-- "API Call Hold / Debit / Credit" --> ACC
            PAY e3@-- "API Call Post Double-entry Journal" --> LED
            PAY e4@-- "API Call Calculate Fees" --> FEE
            PAY e5@-- "Async Event Send Transaction Notification" --> NOTIF
            PAY e6@-- "API Call Transfer Request / Status Inquiry" --> BANK
        
            %% Link styles (indices 0..9 match the edges above)
            linkStyle 0 stroke:#2ecc71, stroke-width:3px;
            linkStyle 1 stroke:#27ae60, stroke-width:2.5px;
            linkStyle 2 stroke:#27ae60, stroke-width:2.5px;
            linkStyle 3 stroke:#27ae60, stroke-width:2.5px;
            linkStyle 4 stroke:#9b59b6, stroke-width:3px;
            linkStyle 5 stroke:#2980b9, stroke-width:3px;
            
            classDef animate stroke-dasharray: 9,5,stroke-dashoffset: 900,animation: dash 25s linear infinite;
            class e1,e2,e3,e4,e5,e6 animate;

            %% Optional simple node styling (most renderers accept this)
            classDef core fill:#2c3e50,stroke:#1abc9c,color:#ecf0f1;
            class CLIENT,PAY,ACC,LED,FEE,NOTIF,BANK,RECON,REPORT core;

    ```