# SmsVoid

SmsVoid es un servicio de captura de mensajes SMPP. Guarda los envíos recibidos en SQLite o PostgreSQL y permite consultarlos mediante un panel HTTP y una API JSON.

## Inicio rápido

```sh
go build -o bin/smsvoid ./cmd/smsvoid
./bin/smsvoid
```

De forma predeterminada, SMPP escucha en `:2775`, HTTP en `:8025` y SQLite utiliza el archivo `smsvoid.db` en el directorio de trabajo. Configure el cliente SMPP para conectarse a `localhost:2775` y luego abra <http://localhost:8025>.

## Configuración

| Bandera | Variable de entorno | Valor predeterminado | Descripción |
|---|---|---|---|
| `-smpp-addr` | `SMSVOID_SMPP_ADDR` | `:2775` | Dirección TCP de escucha SMPP |
| `-http-addr` | `SMSVOID_HTTP_ADDR` | `:8025` | Dirección HTTP de escucha |
| `-db-driver` | `SMSVOID_DB_DRIVER` | `sqlite` | `sqlite` o `postgres` |
| `-db-dsn` | `SMSVOID_DB_DSN` | `smsvoid.db` | Ruta SQLite o DSN PostgreSQL |

Las banderas tienen prioridad sobre las variables de entorno. Ejemplo con PostgreSQL:

```sh
SMSVOID_DB_DRIVER=postgres SMSVOID_DB_DSN='postgres://smsvoid:secreto@localhost:5432/smsvoid?sslmode=require' ./bin/smsvoid
```

SmsVoid crea la tabla de mensajes al iniciar. La base de datos PostgreSQL debe existir y el usuario configurado necesita permiso para crear tablas.

## Comportamiento SMPP

PDUs implementadas: `bind_receiver`, `bind_transmitter`, `bind_transceiver`, `submit_sm`, `enquire_link` y `unbind`. Actualmente se acepta cualquier identificador de sistema y no se autentican credenciales. Un `submit_sm` aceptado recibe un identificador generado y guarda origen, destino, cuerpo, codificación, secuencia, hora de recepción y dirección remota.

El analizador procesa el contenido obligatorio de `short_message`. Aún no procesa `message_payload`, TLV opcionales, concatenación multipartes, recibos de entrega ni mensajes `deliver_sm` iniciados por el servidor. Es una herramienta de captura para desarrollo, no un SMSC de producción.

## HTTP

- `GET /`: panel HTML de capturas.
- `GET /api/v1/messages`: arreglo JSON con hasta 250 mensajes recientes.

La interfaz HTTP no incluye autenticación. Limite el acceso a una red confiable o utilice un proxy autenticado.

## Arquitectura

- `internal/smpp`: adaptador de transporte SMPP y manejo de PDUs.
- `internal/app`: generación de identificadores y caso de uso de envío.
- `internal/storage`: persistencia SQLite/PostgreSQL.
- `internal/web`: panel y adaptador JSON.
- `internal/config`: configuración por entorno y banderas.
- `cmd/smsvoid`: inicio y cierre ordenado del proceso.

## Desarrollo

```sh
go test ./...
go test -race ./...
go test -cover ./...
go build ./cmd/smsvoid
```

Las pruebas SQLite crean bases de datos temporales. Pruebe la integración PostgreSQL contra una base de datos configurada expresamente.
