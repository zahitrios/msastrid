# Astrid

Es un ms para nuestro listado de precios

## Installation

```bash
npm i
go mod download
```

## Usage
Setear variables dentro de ./products/local.env y correr el siguiente comando
```bash
make run
```

## Endpoints
```bash
GET - http://{{host}}/campaigns/{id}
POST - http://{{host}}/campaigns/{id}
DELETE - http://{{host}}/campaigns/{id}
GET - http://{{host}}/campaigns
POST - http://{{host}}/campaigns
GET - http://{{host}}/campaigns/{campaignId}/items
GET - http://{{host}}/campaigns/{campaignId}/items/pendding
POST - http://{{host}}/campaigns/{campaignId}/items/bulk-approve
GET - http://{{host}}/price-list
GET - http://{{host}}/price-list/publish
GET - http://{{host}}/price-list/{sku}
POST - http://{{host}}/price-list
POST - http://{{host}}/price-list/publish
GET - http://{{host}}/gaia-groups
GET - http://{{host}}/gaia-groups/publish
GET - http://{{host}}/gaia-groups/{sku}
POST - http://{{host}}/gaia-groups
DELETE - http://{{host}}/gaia-groups
POST - http://{{host}}/gaia-groups/publish
POST - http://{{host}}/gaia-groups/sync
GET - http://{{host}}/merge-price-report
GET - http://{{host}}/consume-pim-products
GET - http://{{host}}/users/role/{email}
GET - http://{{host}}/export/{collection}
GET - http://{{host}}/logs
```
## Deploy
STG
```bash
make deploy-stg
```

PROD
```bash
make deploy-prod
```