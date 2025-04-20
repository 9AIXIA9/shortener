CREATE DATABASE IF NOT EXISTS shortener;
USE shortener;

CREATE TABLE IF NOT EXISTS `sequence`
(
    `id`        BIGINT UNSIGNED NOT NULL,
    `stub`      CHAR(1)         NOT NULL DEFAULT '0'
        COMMENT '占位符',
    `timestamp` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_stub` (`stub`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 COMMENT ='序号表';

INSERT INTO sequence(id,stub) VALUES (0,'a');