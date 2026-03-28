create table if not exists file_metadata (
    id uuid primary key,
    object_key text not null unique,
    original_filename text not null,
    mime_type text not null,
    size bigint not null check (size >= 0),
    uploader text,
    owner_service text,
    created_at timestamptz not null default now(),
    deleted_at timestamptz
);

create index if not exists idx_file_metadata_created_at on file_metadata(created_at desc);
create index if not exists idx_file_metadata_uploader on file_metadata(uploader);
create index if not exists idx_file_metadata_owner_service on file_metadata(owner_service);
create index if not exists idx_file_metadata_deleted_at on file_metadata(deleted_at);
