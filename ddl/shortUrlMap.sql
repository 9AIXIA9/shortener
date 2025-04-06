CREATE TABLE IF NOT EXISTS `short_url_map`
(
    `id`        BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `create_at` TIMESTAMP        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `create_by` VARCHAR(64)      NOT NULL DEFAULT 'system' COMMENT '创建者',
    `update_at` TIMESTAMP        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `is_del`    TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否删除：0正常1删除',
    `long_url`  VARCHAR(2048)    NOT NULL DEFAULT '' COMMENT '长链接',
    `md5`       CHAR(32)         NOT NULL DEFAULT '' COMMENT '长链接MD5',
    `short_url` VARCHAR(11)      NOT NULL DEFAULT '' COMMENT '短链接',
    PRIMARY KEY (`id`),
    INDEX `idx_is_del` (`is_del`),
    INDEX `idx_create_at` (`create_at`),
    UNIQUE `uniq_md5` (`md5`),
    UNIQUE `uniq_short_url` (`short_url`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='长短链映射表';