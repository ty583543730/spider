##terry的腾讯课堂爬虫ReadMe

###提前准备项

*1：访问http需要提供md5加密后密文*
使用4534253453252345353534253453245342data_str4534253453252345353534253453245342identifierterry  进行MD5加密即可
以下三个参数必填，然后填写查询参数
入参格式如下
{"Identifier":"terry","signature":"bab71ca3bc9488e30d6ab9b0fcde4418","data_str":"4534253453252345353534253453245342"}

数据库表设计

/*
Navicat MySQL Data Transfer

Source Server         : terry
Source Server Version : 50553
Source Host           : localhost:3306
Source Database       : test

Target Server Type    : MYSQL
Target Server Version : 50553
File Encoding         : 65001

Date: 2020-03-08 16:36:51
*/

SET FOREIGN_KEY_CHECKS=0;

-- ----------------------------
-- Table structure for courses
-- ----------------------------
DROP TABLE IF EXISTS `courses`;
CREATE TABLE `courses` (
  `id` int(11) unsigned zerofill NOT NULL AUTO_INCREMENT,
  `course_id` int(11) unsigned NOT NULL,
  `url` varchar(512) NOT NULL,
  `father_title` varchar(512) NOT NULL,
  `subject` varchar(255) NOT NULL,
  `time` varchar(255) NOT NULL,
  `teacher_name` varchar(255) NOT NULL,
  `price` varchar(64) NOT NULL DEFAULT '0',
  `child_title` varchar(255) NOT NULL,
  `bg_time` int(11) NOT NULL,
  `teacher_name_detail` varchar(64) NOT NULL,
  `add_time` int(11) NOT NULL,
  `last_modify_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `bg_time` (`bg_time`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1607 DEFAULT CHARSET=utf8;


爬取整体流程：

1、递归爬取主站所有子链接，加入bitmap和链接列表
2、bitmap去重
3、解析链接选出含有课程的信息的链接
4、获取到课程链接然后正则匹配对应的信息
5、将匹配到的信息转化成结构体数组数据
6、将数据存入db 

web服务流程

1、运行开启服务
2、接收post请求，并根据参数鉴权
3、获取参数查询近n天的数据
4、返回前端并渲染展示

访问地址:http://104.225.233.77/show.html