# Create Database: {{ database }}
# ------------------------------------------------------------

CREATE DATABASE IF NOT EXIST `{{ database }}` DEFAULT CHARSET {{ charset }} COLLATE {{ collate }};
USE `{{ database }}` ;
