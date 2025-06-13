create table stories (
    id int generated always as identity,
    story_id int not null,
    api_version char(2) not null,
    queue_name text not null,
    fetched_at timestamp without time zone not null,
    raw_document jsonb not null,

    primary key (id),
    unique (story_id, queue_name)
);

-- Only top level comments on stories.
create table comments (
    id int generated always as identity,
    internal_story_id int not null,
    comment_id int not null,
    raw_document jsonb not null,

    primary key (id),
    foreign key (internal_story_id) references stories (id),
    unique (internal_story_id, comment_id)
);
