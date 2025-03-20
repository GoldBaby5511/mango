-- 创建逻辑库
DROP DATABASE if EXISTS mango_property;
CREATE DATABASE mango_property DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

/*----------------------------------------------------------------------------------*/
-- 财富信息
DROP TABLE IF EXISTS `mango_property`.`wealth`;
CREATE TABLE `mango_property`.`wealth` (
  `sys_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增id',
  `user_id` BIGINT NOT NULL DEFAULT '0' COMMENT 'user_id',
  `ingot` BIGINT NOT NULL DEFAULT '0' COMMENT 'ingot',
  `coin` BIGINT NOT NULL DEFAULT '0' COMMENT 'coin',
  `last_change_id` SMALLINT UNSIGNED NOT NULL DEFAULT '0' COMMENT '最后变化财富id<=100',
  `last_change_count` BIGINT NOT NULL DEFAULT '0' COMMENT '最后变化数量',
  `last_change_reason` INT NOT NULL DEFAULT '0' COMMENT '最后变化原因',
  `last_change_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后变化时间',
  PRIMARY KEY (`sys_id`,`user_id`) USING BTREE,
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='财富信息';


/*----------------------------------------------------------------------------------*/
-- 财富变化日志
DROP TABLE IF EXISTS `mango_property`.`wealth_change_log`;
CREATE TABLE `mango_property`.`wealth_change_log` (
  `sys_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增id',
  `user_id` BIGINT NOT NULL DEFAULT '0' COMMENT 'user_id',
  `reason` INT NOT NULL DEFAULT '0' COMMENT '原因',
  `source_app_type` INT NOT NULL DEFAULT '0' COMMENT '来源类型',
  `source_app_id` INT NOT NULL DEFAULT '0' COMMENT '来源id',
  `change_id` SMALLINT UNSIGNED NOT NULL DEFAULT '0' COMMENT '变化财富id<=100',
  `change_count` BIGINT NOT NULL DEFAULT '0' COMMENT '变化数量',
  `change_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '变化时间',
  PRIMARY KEY (`sys_id`,`user_id`) USING BTREE,
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='财富变化日志';

/*----------------------------------------------------------------------------------*/
-- 道具信息
DROP TABLE IF EXISTS `mango_property`.`prop`;
CREATE TABLE `mango_property`.`prop` (
  `sys_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增id',
  `user_id` BIGINT NOT NULL DEFAULT '0' COMMENT 'user_id',
  `id` BIGINT UNSIGNED NOT NULL DEFAULT '0' COMMENT '道具id',
  `count` BIGINT NOT NULL DEFAULT '0' COMMENT '道具数量',
  `last_change_count` BIGINT NOT NULL DEFAULT '0' COMMENT '最后变化数量',
  `last_change_reason` INT NOT NULL DEFAULT '0' COMMENT '最后变化原因',
  `last_change_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后变化时间',
  PRIMARY KEY (`sys_id`,`user_id`) USING BTREE,
  KEY `idx_user_id` (`user_id`),
  KEY `idx_id` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='道具信息';

/*----------------------------------------------------------------------------------*/
-- 道具变化日志
DROP TABLE IF EXISTS `mango_property`.`prop_change_log`;
CREATE TABLE `mango_property`.`prop_change_log` (
  `sys_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增id',
  `user_id` BIGINT NOT NULL DEFAULT '0' COMMENT 'user_id',
  `reason` INT NOT NULL DEFAULT '0' COMMENT '原因',
  `source_app_type` INT NOT NULL DEFAULT '0' COMMENT '来源类型',
  `source_app_id` INT NOT NULL DEFAULT '0' COMMENT '来源id',
  `change_id` BIGINT UNSIGNED NOT NULL DEFAULT '0' COMMENT '变化道具',
  `change_count` BIGINT NOT NULL DEFAULT '0' COMMENT '变化数量',
  `change_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '变化时间',
  PRIMARY KEY (`sys_id`,`user_id`) USING BTREE,
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='道具变化日志';