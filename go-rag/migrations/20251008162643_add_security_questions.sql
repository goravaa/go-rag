-- Create "security_questions" table
CREATE TABLE "security_questions" (
  "id" uuid NOT NULL,
  "question" character varying NOT NULL,
  "answer" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "user_security_questions" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "security_questions_users_security_questions" FOREIGN KEY ("user_security_questions") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
