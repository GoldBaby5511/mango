-- 创建逻辑库
DROP DATABASE if EXISTS mango_logic;
CREATE DATABASE mango_logic DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

/*----------------------------------------------------------------------------------*/
/*创建表*/
/*----------------------------------------------------------------------------------*/
-- 账号信息
DROP TABLE IF EXISTS `mango_logic`.`account`;
CREATE TABLE `mango_logic`.`account` (
  `sys_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增id',
  `user_id` BIGINT UNSIGNED NOT NULL DEFAULT '0' COMMENT 'user_id',
  `game_id` BIGINT NOT NULL DEFAULT '0' COMMENT 'game_id',
  `account` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '账号',
  `pass_word` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '密码',
  `nick_name` VARCHAR(30) NOT NULL DEFAULT '' COMMENT '昵称',
  `avatar` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '头像',
  `sex` INT NOT NULL DEFAULT '0' COMMENT '性别 1、男 0、女',
  `active` INT NOT NULL DEFAULT '0' COMMENT '用户累计活跃天数',
  `age` INT NOT NULL DEFAULT '0',
  `login_type` INT NOT NULL DEFAULT '0' COMMENT '类型 0=账号 1=Token 2=唯一标识登录',
  `device_id` VARCHAR(256) NOT NULL DEFAULT '' COMMENT '设备唯一ID',
  `reg_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '注册时间',
  `reg_ip` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '注册IP地址',
  `os` INT NOT NULL DEFAULT '0' COMMENT '0、未知 1、IOS 2、Android 3、h5',
  `frozen` INT NOT NULL DEFAULT '0' COMMENT '冻结状态 1、冻结 0、正常',
  `token` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'token',
  `market_id` INT NOT NULL DEFAULT '0' COMMENT '登录主渠道',
  `site_id` INT NOT NULL DEFAULT '0' COMMENT '登录子渠道',
  `reg_market_id` INT NOT NULL DEFAULT '0' COMMENT '注册主渠道',
  `reg_site_id` INT NOT NULL DEFAULT '0' COMMENT '注册子渠道',
  `user_type` TINYINT NOT NULL DEFAULT '0' COMMENT '用户类型',
  `last_login_time` TIMESTAMP NULL DEFAULT NULL COMMENT '最后登录时间',
  `last_login_type` INT NOT NULL DEFAULT '0' COMMENT '最后登录类型',
  `last_login_ip` VARCHAR(128) NOT NULL DEFAULT '',
  `last_login_addr` VARCHAR(128) NOT NULL DEFAULT '',
  `last_login_area` VARCHAR(128) NOT NULL DEFAULT '',
  PRIMARY KEY (`sys_id`,`user_id`) USING BTREE,
  UNIQUE KEY `user_id` (`user_id`),
  UNIQUE KEY `game_id` (`game_id`),
  KEY `nick_name` (`nick_name`),
  KEY `device_id` (`device_id`),
  KEY `login_type` (`login_type`),
  KEY `user_type` (`user_type`),
  KEY `ix_lastlogintime` (`last_login_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='账号信息';

/*----------------------------------------------------------------------------------*/
-- 登录日志
DROP TABLE IF EXISTS `mango_logic`.`login_log`;
CREATE TABLE `mango_logic`.`login_log` (
  `sys_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增id',
  `user_id` BIGINT UNSIGNED NOT NULL DEFAULT '0' COMMENT 'user_id',
  `game_id` BIGINT NOT NULL DEFAULT '0' COMMENT 'game_id',
  `account` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '账号',
  `reason` INT NOT NULL DEFAULT '0' COMMENT '原因',
  `device_id` VARCHAR(256) NOT NULL DEFAULT '' COMMENT '设备唯一ID',
  `login_time` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '登录时间',
  `logout_time` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '退出时间',
  PRIMARY KEY (`sys_id`,`user_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='登录日志';

/*----------------------------------------------------------------------------------*/

-- 新手引导
DROP TABLE IF EXISTS `mango_logic`.`novice_guide`;
CREATE TABLE `mango_logic`.`novice_guide` (
  `sys_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增id',
  `user_id` BIGINT UNSIGNED NOT NULL DEFAULT '0' COMMENT '玩家id',
  `step_number` INT UNSIGNED NOT NULL DEFAULT '0' COMMENT '进度值',
  `create_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `change_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '变化时间',
  PRIMARY KEY (`sys_id`) USING BTREE,
  UNIQUE `idx_user_id` (`user_id`),
  KEY `idx_step_number` (`step_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='新手引导';