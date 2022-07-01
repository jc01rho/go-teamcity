FROM jetbrains/teamcity-server:2021.2.3

COPY reset.sh /reset.sh
CMD ["/reset.sh"]