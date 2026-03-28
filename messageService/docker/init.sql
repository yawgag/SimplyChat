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
    kind text not null default 'text' check (kind in ('text', 'file', 'system')),
    message_text text, -- nullable
    added_at timestamptz not null default now()
);

create index idx_message_chat_added_id_desc on message(chat_id, added_at desc, id desc);

create table message_attachment(
    id serial primary key,
    message_id int not null references message(id) on delete cascade,
    file_id uuid not null,
    original_filename text not null,
    mime_type text not null,
    size bigint not null check (size >= 0),
    kind text not null check (kind in ('file', 'image')),
    created_at timestamptz not null default now(),
    unique(message_id, file_id)
);

create index idx_message_attachment_message_id on message_attachment(message_id);
create index idx_message_attachment_file_id on message_attachment(file_id);

/*
data: json{
    type TEXT (text, voice, photo, video, unknown),
    data TEXT 
}
*/
