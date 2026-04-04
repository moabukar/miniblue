# Terraform Scenarios

Example architectures you can test locally with miniblue.

## Scenarios

| Scenario | Description | Resources |
|----------|-------------|-----------|
| [three-tier](three-tier/) | Web + App + Data tier with VNet isolation | RG, VNet, 3 Subnets, DNS, ACR |
| [serverless](serverless/) | Event-driven with Functions and Event Grid | RG, Event Grid, DNS |
| [microservices](microservices/) | Multi-service with shared infra and per-service subnets | 4 RGs, VNet, 3 Subnets, ACR, DNS, Event Grid |

## Running any scenario

```bash
# 1. Start miniblue
./bin/miniblue

# 2. Trust the cert
export SSL_CERT_FILE=~/.miniblue/cert.pem

# 3. Run the scenario
cd examples/terraform/scenarios/three-tier
terraform init
terraform apply -auto-approve

# 4. Verify with azlocal
azlocal group list
azlocal health

# 5. Tear down
terraform destroy -auto-approve
```
