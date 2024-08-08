-- CREATE TABLE IF NOT EXISTS events
-- (
--     event_id     UUID PRIMARY KEY,
--     order_id     UUID         NOT NULL,
--     user_id      UUID         NOT NULL,
--     order_status VARCHAR(255) NOT NULL,
--     updated_at   TIMESTAMP    NOT NULL,
--     created_at   TIMESTAMP    NOT NULL
-- );


CREATE TABLE IF NOT EXISTS orders
(
    order_id     UUID         PRIMARY KEY,
    user_id      UUID         NOT NULL,
    order_status VARCHAR(255) NOT NULL,
    is_final     BOOLEAN NOT NULL,
    updated_at   TIMESTAMP    NOT NULL,
    created_at   TIMESTAMP    NOT NULL
);


CREATE TABLE IF NOT EXISTS order_events
(
    event_id     UUID PRIMARY KEY,
    order_id     UUID         NOT NULL,
    user_id      UUID         NOT NULL,
    order_status VARCHAR(255) NOT NULL,
    is_final     BOOLEAN NOT NULL,
    updated_at   TIMESTAMP    NOT NULL,
    created_at   TIMESTAMP    NOT NULL,
    is_in_order  BOOLEAN      NOT NULL DEFAULT FALSE
);