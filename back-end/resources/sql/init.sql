CREATE TABLE room (
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

CREATE TABLE user_at_call (
    room_id VARCHAR(36) REFERENCES room(room_id) ON DELETE CASCADE,
    user_id VARCHAR(36),
    shown_name VARCHAR(100),
    is_admin BOOLEAN DEFAULT FALSE
);

CREATE TABLE registered_user (
    user_id VARCHAR(36) PRIMARY KEY,
    phone VARCHAR(20),
    email VARCHAR(40),
    password VARCHAR(256) not null,
    shown_name VARCHAR(100),
    role integer REFERENCES roles(role_id)
);

CREATE TABLE refresh_token (
    token_id SERIAL PRIMARY KEY,
    content VARCHAR(100) NOT NULL UNIQUE,
    expires_at TIMESTAMP,
    user_id VARCHAR(36) REFERENCES registered_user(user_id)
);

CREATE TABLE roles (
    role_id serial PRIMARY KEY,
    name VARCHAR(10)
);

INSERT INTO roles(name) VALUES ('USER'),('ADMIN');

insert into registered_user(user_id, phone, email, password) values('1', '1234', 'test@test.com', 'hello world');
insert into room values('1', now(), null, 'title', 'description', '1');

-- Получение комнаты по id
SELECT 
    created_at, 
    starts_at, 
    title, 
    description, 
    (SELECT COUNT(*) FROM user_at_call WHERE room_id='1') as users_count, 
    shown_name as creator
from room r JOIN registered_user ru ON r.owner_id = ru.user_id where room_id='1';  
-- Получение комнаты по user_id
SELECT 
    room_id,
    created_at, 
    starts_at, 
    title, 
    description, 
    (SELECT COUNT(*) FROM user_at_call WHERE room_id=r.room_id) as users_count
from room r JOIN registered_user ru ON r.owner_id = ru.user_id where owner_id='1'; 
SELECT room_id, created_at, start_at, title, description, user_count;

SELECT Count(*) from registered_user WHERE (email IS NOT NULL AND email='test') OR (phone IS NOT NULL AND phone = 'test'); 

