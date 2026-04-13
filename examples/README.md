# miniblue Examples

Code examples showing how to use miniblue with different languages and tools.

## Examples

| Example | Language | What it shows |
|---------|----------|--------------|
| [python/](python/) | Python | Azure SDK for Python with resource groups |
| [go/](go/) | Go | HTTP client for resource groups and Key Vault |
| [javascript/](javascript/) | JavaScript | Fetch API for resource groups, Key Vault, Blob Storage |
| [dotnet/](dotnet/) | C# / .NET 10 for .NET with resource groups, Key Vault, Blob Storage, Cosmos DB |
| [terraform/](terraform/) | HCL | Full Terraform azurerm provider setup with 5 resource types |
| [ci/](ci/) | YAML | GitHub Actions workflow with miniblue as a service container |

## Running examples

Start miniblue first:

    ./bin/miniblue

Then run any example:

    cd examples/python && pip install -r requirements.txt && python example.py
    cd examples/go && go run main.go
    cd examples/javascript && node example.js
    cd examples/dotnet && dotnet run
    cd examples/terraform && export SSL_CERT_FILE=~/.miniblue/cert.pem && terraform init && terraform apply
