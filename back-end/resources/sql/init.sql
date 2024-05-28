CREATE DATABASE meetservice;

CREATE TABLE meetservice.room (
    room_id VARCHAR(36) PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    starts_at TIMESTAMP,
    title VARCHAR(255),
    description VARCHAR(255),
    owner_id VARCHAR(36) REFERENCES registered_user(user_id),
    active boolean DEFAULT false,
    last_interact TIMESTAMP,
    call_start TIMESTAMP
);

CREATE TABLE meetservice.user_at_call (
    room_id VARCHAR(36) REFERENCES room(room_id) ON DELETE CASCADE,
    user_id VARCHAR(36),
    shown_name VARCHAR(100),
    is_admin BOOLEAN DEFAULT FALSE
);

CREATE TABLE meetservice.registered_user (
    user_id VARCHAR(36) PRIMARY KEY,
    phone VARCHAR(20),
    email VARCHAR(40),
    password VARCHAR(256) not null,
    shown_name VARCHAR(100),
    role integer REFERENCES roles(role_id)
);

CREATE TABLE meetservice.refresh_token (
    token_id SERIAL PRIMARY KEY,
    content VARCHAR(100) NOT NULL UNIQUE,
    expires_at TIMESTAMP,
    user_id VARCHAR(36) REFERENCES registered_user(user_id)
);

CREATE TABLE meetservice.roles (
    role_id serial PRIMARY KEY,
    name VARCHAR(10)
);
