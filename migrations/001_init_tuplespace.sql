CREATE TABLE IF NOT EXISTS tuples (
    id          BIGSERIAL PRIMARY KEY,
    space       TEXT NOT NULL,
    arity       INT NOT NULL,
    fields_json JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tuple_fields (
    tuple_id    BIGINT NOT NULL REFERENCES tuples(id) ON DELETE CASCADE,
    pos         INT NOT NULL,
    type        TEXT NOT NULL,
    text_val    TEXT,
    int_val     BIGINT,
    bool_val    BOOLEAN,
    PRIMARY KEY (tuple_id, pos)
);

CREATE INDEX IF NOT EXISTS tuples_space_arity_id_idx
    ON tuples(space, arity, id);

CREATE INDEX IF NOT EXISTS tuple_fields_text_idx
    ON tuple_fields(pos, type, text_val, tuple_id);

CREATE INDEX IF NOT EXISTS tuple_fields_int_idx
    ON tuple_fields(pos, type, int_val, tuple_id);

CREATE INDEX IF NOT EXISTS tuple_fields_bool_idx
    ON tuple_fields(pos, type, bool_val, tuple_id);

