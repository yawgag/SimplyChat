create table chat(
    id serial primary key,
    maxMembers int not null
    -- chatName text not null
);

create table chat_user(
    chatId int not null references chat(id) on delete cascade, 
    userLogin text not null,
    primary key (chatId, userLogin)
);

create table message(
    id serial primary key,
    chatId int not null references chat(id) on delete cascade,
    senderLogin text not null,
    data text not null,
    addedAt TIMESTAMPTZ not null default now()
);