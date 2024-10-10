DROP SCHEMA IF EXISTS transit;
CREATE SCHEMA transit;
USE transit;

DROP TABLE IF EXISTS `operations`;
CREATE TABLE `operations` (
  `train_id` int unsigned NOT NULL,
  `op_order` int unsigned NOT NULL,
  `dep_sta_id` int unsigned NOT NULL,
  `dep_time` time NOT NULL,
  `arr_sta_id` int unsigned NOT NULL,
  `arr_time` time NOT NULL,
  PRIMARY KEY (`train_id`,`op_order`),
  KEY `operations_stations_FK` (`dep_sta_id`),
  KEY `operations_stations_FK_1` (`arr_sta_id`),
  CONSTRAINT `operations_stations_FK` FOREIGN KEY (`dep_sta_id`) REFERENCES `stations` (`id`),
  CONSTRAINT `operations_stations_FK_1` FOREIGN KEY (`arr_sta_id`) REFERENCES `stations` (`id`),
  CONSTRAINT `operations_trains_FK` FOREIGN KEY (`train_id`) REFERENCES `trains` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `stations`;
CREATE TABLE `stations` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  `name_en` varchar(100) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=103 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `trains`;
CREATE TABLE `trains` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `trains_unique` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=102 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
