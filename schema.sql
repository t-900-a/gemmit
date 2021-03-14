DROP TABLE submissions;
DROP TABLE payments;
DROP TABLE entries;
DROP TABLE feeds;
DROP TABLE accepted_payments;
DROP TABLE authors;
DROP TYPE feed_kind;
DROP TABLE users;

CREATE TABLE users (
                       id serial PRIMARY KEY,
                       created timestamp NOT NULL,
                       certhash varchar(128) NOT NULL UNIQUE
);

CREATE TYPE feed_kind AS ENUM ('gemini', 'rss');

CREATE TABLE authors (
                         id serial PRIMARY KEY,
                         name varchar,
                         created timestamp NOT NULL,
                         updated timestamp NOT NULL,
                         url varchar,
                         email varchar
);

CREATE TABLE accepted_payments (
                                   id serial PRIMARY KEY,
                                   author_id INTEGER NOT NULL references authors(id),
                                   pay_type varchar NOT NULL,
                                   view_key varchar UNIQUE,
                                   address varchar UNIQUE,
                                   registered BOOLEAN NOT NULL,
                                   UNIQUE (author_id, id)
);

CREATE TABLE feeds (
                       id serial PRIMARY KEY,
                       created timestamp NOT NULL,
                       updated timestamp NOT NULL,
                       author_id INTEGER NOT NULL references authors(id),
                       kind feed_kind NOT NULL,
                       url varchar UNIQUE,
                       title varchar,
                       description varchar
);

CREATE TABLE entries (
                          id serial PRIMARY KEY,
                          title varchar NOT NULL,
                          published timestamp NOT NULL,
                          url varchar NOT NULL,
                          feed_id INTEGER NOT NULL references feeds(id),
                          UNIQUE (url, feed_id)
);

CREATE TABLE payments (
                                 id serial PRIMARY KEY,
                                 pay_type varchar NOT NULL,
                                 address varchar NOT NULL,
                                 tx_id varchar NOT NULL,
                                 tx_date timestamp NOT NULL,
                                 amount decimal NOT NULL,
                                 accepted_payments_id INTEGER NOT NULL references accepted_payments(id)
);

CREATE TABLE submissions (
                               id serial PRIMARY KEY,
                               user_id INTEGER NOT NULL references users(id),
                               feed_id INTEGER NOT NULL references feeds(id),
                               UNIQUE (user_id, feed_id)
);
