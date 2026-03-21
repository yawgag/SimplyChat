create table chat(
    id serial primary key,
    max_members int not null
    -- chatName text not null
);

create table chat_user(
    chat_id int not null references chat(id) on delete cascade, 
    user_login text not null,
    primary key (chat_id, user_login)
);

create index idx_chat_user_by_user on chat_user(user_login);

create table message(
    id serial primary key,
    chat_id int not null references chat(id) on delete cascade,
    sender_login text not null,
    -- type text not null, -- text | media | system
    message_text text, -- nullable
    added_at timestamptz not null default now()
);

create index idx_message_chat_timestamp on message(chat_id, added_at desc);

/*
data: json{
    type TEXT (text, voice, photo, video, unknown),
    data TEXT 
}
*/