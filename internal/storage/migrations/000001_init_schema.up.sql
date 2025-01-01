CREATE TABLE users
(
    id uuid PRIMARY KEY,
    login varchar NOT NULL UNIQUE,
    password varchar NOT NULL
);

CREATE TYPE status AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');

CREATE TABLE orders
(
    id serial PRIMARY KEY,
    number varchar NOT NULL UNIQUE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    uploaded_at timestamptz DEFAULT NOW(),
    status_processing status NOT NULL,
    accrual integer DEFAULT 0
); 


CREATE TABLE withdraws
(
    id serial PRIMARY KEY,
    withdraw_number varchar NOT NULL,
    sum integer NOT NULL,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    processed_at timestamptz DEFAULT NOW()
);