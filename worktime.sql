-- phpMyAdmin SQL Dump
-- version 4.6.4
-- https://www.phpmyadmin.net/
--
-- Host: localhost
-- Generation Time: Sep 05, 2016 at 01:27 PM
-- Server version: 10.1.16-MariaDB
-- PHP Version: 7.0.10

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
SET time_zone = "+00:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

--
-- Database: `worktime`
--

-- --------------------------------------------------------

--
-- Table structure for table `groups`
--

CREATE TABLE `groups` (
  `groupid` varchar(100) NOT NULL,
  `groupname` varchar(100) NOT NULL
) ENGINE=MyISAM DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- Table structure for table `users`
--

CREATE TABLE `users` (
  `epassport` varchar(100) NOT NULL COMMENT 'epassport id',
  `name` varchar(100) NOT NULL,
  `surname` varchar(100) NOT NULL,
  `groupid` varchar(100) NOT NULL,
  `active` tinyint(1) NOT NULL DEFAULT '1'
) ENGINE=MyISAM DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- Table structure for table `worktime`
--

CREATE TABLE `worktime` (
  `uid` bigint(20) NOT NULL,
  `epassport` varchar(100) NOT NULL,
  `timex` datetime NOT NULL,
  `ip` varchar(100) NOT NULL,
  `pic` longtext NOT NULL
) ENGINE=MyISAM DEFAULT CHARSET=utf8;

--
-- Indexes for dumped tables
--

--
-- Indexes for table `groups`
--
ALTER TABLE `groups`
  ADD PRIMARY KEY (`groupid`);

--
-- Indexes for table `users`
--
ALTER TABLE `users`
  ADD PRIMARY KEY (`epassport`);

--
-- Indexes for table `worktime`
--
ALTER TABLE `worktime`
  ADD PRIMARY KEY (`uid`);


