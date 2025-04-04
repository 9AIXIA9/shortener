CREATE TABLE IF NOT EXISTS `sequence`
(
    `id`        BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `stub`      CHAR(1)         NOT NULL DEFAULT '0'
        COMMENT '占位符',
#通过唯一性约束确保表中仅存在一行数据
# 利用MySQL的 REPLACE INTO 或
# INSERT ... ON DUPLICATE KEY UPDATE
# 操作触发自增ID生成
    `timestamp` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_stub` (`stub`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8 COMMENT ='序号表';