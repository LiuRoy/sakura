CREATE TABLE answer (
        id INTEGER NOT NULL,
        question_id INTEGER NOT NULL,
        answer_id INTEGER NOT NULL,
        question TEXT,
        answer TEXT,
        star INTEGER NOT NULL,
        PRIMARY KEY (id)
);
CREATE INDEX ix_question on answer(question_id);
CREATE INDEX ix_answer on answer(answer_id);

CREATE TABLE label (
        id INTEGER NOT NULL,
        question_id INTEGER NOT NULL,
        label VARCHAR(50),
        PRIMARY KEY (id)
)
