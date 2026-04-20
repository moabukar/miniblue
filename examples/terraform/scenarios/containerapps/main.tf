# Container Apps Architecture on miniblue
# Managed Environments + Container Apps + Jobs
#
# Usage:
#   export SSL_CERT_FILE=~/.miniblue/cert.pem
#   terraform init && terraform apply -auto-approve
#
# Test with azlocal after apply:
#   azlocal group list
#   azlocal managed-environment show --name apps-env --resource-group containerapps-rg
#   azlocal container-app show --name api-app --resource-group containerapps-rg
#   azlocal container-app show --name web-app --resource-group containerapps-rg
#   azlocal container-app-job show --name batch-job --resource-group containerapps-rg
#   azlocal container-app revision list --name api-app --resource-group containerapps-rg
#
# Destroy:
#   terraform destroy -auto-approve

# --- Foundation ---

resource "azurerm_resource_group" "main" {
  name     = "containerapps-rg"
  location = "East US"
}

# --- Managed Environment ---

resource "azurerm_container_app_environment" "apps" {
  name                = "apps-env"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
}

# --- Container Apps ---

resource "azurerm_container_app" "api" {
  name                         = "api-app"
  resource_group_name          = azurerm_resource_group.main.name
  container_app_environment_id = azurerm_container_app_environment.apps.id
  revision_mode                = "Single"

  ingress {
    target_port      = 8080
    external_enabled = true
    transport        = "http"

    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
  }

  template {
    container {
      name   = "api"
      image  = "mcr.microsoft.com/azuredocs/containerapps-helloworld:latest"
      cpu    = 0.25
      memory = "0.5Gi"
    }
  }

  secret {
    name  = "api-key"
    value = "placeholder-secret-value"
  }
}

resource "azurerm_container_app" "web" {
  name                         = "web-app"
  resource_group_name          = azurerm_resource_group.main.name
  container_app_environment_id = azurerm_container_app_environment.apps.id
  revision_mode                = "Multiple"

  ingress {
    target_port      = 80
    external_enabled = true
    transport        = "http"

    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
  }

  template {
    min_replicas = 1
    max_replicas = 3

    container {
      name   = "web"
      image  = "mcr.microsoft.com/azuredocs/containerapps-helloworld:latest"
      cpu    = 0.5
      memory = "1Gi"

      env {
        name  = "API_URL"
        value = "https://api-app.hashed.eastus.containerApps.k4apps.io"
      }
    }
  }

  secret {
    name  = "db-password"
    value = "placeholder-secret-value"
  }
}

# --- Container Apps Jobs ---

resource "azurerm_container_app_job" "batch" {
  name                         = "batch-job"
  resource_group_name          = azurerm_resource_group.main.name
  container_app_environment_id = azurerm_container_app_environment.apps.id
  location                     = azurerm_resource_group.main.location

  replica_timeout_in_seconds = 600
  replica_retry_limit        = 3

  manual_trigger_config {
    parallelism = 1
    replica_completion_count = 1
  }

  template {
    container {
      name   = "batch"
      image  = "mcr.microsoft.com/samples/containerapps-batchprocessor:latest"
      cpu    = 1.0
      memory = "2Gi"
    }
  }
}
