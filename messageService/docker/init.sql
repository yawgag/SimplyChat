create table chat(
    id serial primary key,
    userA UUID not null,
    userB UUID not null,
    UNIQUE(userA, userB)
);

create table message(
    id serial primary key,
    chatId int not null references chat(id) on delete cascade,
    senderId UUID not null,
    recipientId UUID not null,
    data text not null,
    addedAt TIMESTAMPTZ not null default now()
);