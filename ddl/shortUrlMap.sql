CREATE TABLE IF NOT EXISTS `short_url_map`
(
    `id`        BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT COMMENT '主键',
    `create_at` DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `create_by` VARCHAR(64)      NOT NULL DEFAULT 'system' COMMENT '创建者',
    `is_del`    TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否删除：0正常1删除',
    `lurl`      VARCHAR(2048)    NOT NULL DEFAULT '' COMMENT '长链接',
    `md5`       CHAR(32)         NOT NULL DEFAULT '' COMMENT '长链接MD5',
    `surl`      VARCHAR(11)      NOT NULL DEFAULT '' COMMENT '短链接',
    PRIMARY KEY (`id`),
    INDEX `idx_is_del` (`is_del`),
    UNIQUE `uniq_md5` (`md5`),
    UNIQUE `uniq_surl` (`surl`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='长短链映射表';